package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/appautoscaling"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	ecrx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	ecsx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecs"
	lbx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")

		containerPort := 80
		if param := cfg.GetInt("containerPort"); param != 0 {
			containerPort = param
		}

		cpu := 512
		if param := cfg.GetInt("cpu"); param != 0 {
			cpu = param
		}

		memory := 1024
		if param := cfg.GetInt("memory"); param != 0 {
			memory = param
		}

		// An ECR repository to store our application's container image
		repo, err := ecrx.NewRepository(ctx, "alprazolam-repo", &ecrx.RepositoryArgs{
			ForceDelete: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Build and publish our application's container image from ./app to the ECR repository
		image, err := ecrx.NewImage(ctx, "alprazolam-img", &ecr.ImageArgs{
			RepositoryUrl: repo.Url,
			Context:       pulumi.String("./app"),
			Platform:      pulumi.String("linux/amd64"),
		})
		if err != nil {
			return err
		}

		// An ECS cluster to deploy into
		cluster, err := ecs.NewCluster(ctx, "alprazolam-cluster", nil)
		if err != nil {
			return err
		}

		// An ALB to serve the container endpoint to the internet
		loadbalancer, err := lbx.NewApplicationLoadBalancer(ctx, "alprazolam-lb", nil)
		if err != nil {
			return err
		}

		// Deploy an ECS Service on Fargate to host the application container
		service, err := ecsx.NewFargateService(ctx, "alprazolam-svc", &ecsx.FargateServiceArgs{
			Cluster:        cluster.Arn,
			AssignPublicIp: pulumi.Bool(true),
			DesiredCount:   pulumi.Int(2),
			// ForceNewDeployment: pulumi.Bool(true), // NOTE: Force a new deployment on every update
			TaskDefinitionArgs: &ecsx.FargateServiceTaskDefinitionArgs{
				Container: &ecsx.TaskDefinitionContainerDefinitionArgs{
					Name:      pulumi.String("alprazolam"),
					Image:     image.ImageUri,
					Cpu:       pulumi.Int(cpu),
					Memory:    pulumi.Int(memory),
					Essential: pulumi.Bool(true),
					PortMappings: ecsx.TaskDefinitionPortMappingArray{
						&ecsx.TaskDefinitionPortMappingArgs{
							ContainerPort: pulumi.Int(containerPort),
							TargetGroup:   loadbalancer.DefaultTargetGroup,
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// Define our auto-scaling target.
		_, err = appautoscaling.NewTarget(ctx, "alprazolam-autoscaling-target", &appautoscaling.TargetArgs{
			ResourceId:        pulumi.StringOutput(pulumi.Sprintf("service/%s/%s", cluster.Name, service.Service.Name())),
			ServiceNamespace:  pulumi.String("ecs"),
			ScalableDimension: pulumi.String("ecs:service:DesiredCount"),
			MaxCapacity:       pulumi.Int(3),
			MinCapacity:       pulumi.Int(1),
		})
		if err != nil {
			return err
		}

		// Define our auto-scaling policy.
		_, err = appautoscaling.NewPolicy(ctx, "alprazolam-autoscaling-policy", &appautoscaling.PolicyArgs{
			ResourceId:        pulumi.StringOutput(pulumi.Sprintf("service/%s/%s", cluster.Name, service.Service.Name())),
			ServiceNamespace:  pulumi.String("ecs"),
			ScalableDimension: pulumi.String("ecs:service:DesiredCount"),
			PolicyType:        pulumi.String("TargetTrackingScaling"),
			TargetTrackingScalingPolicyConfiguration: &appautoscaling.PolicyTargetTrackingScalingPolicyConfigurationArgs{
				PredefinedMetricSpecification: &appautoscaling.PolicyTargetTrackingScalingPolicyConfigurationPredefinedMetricSpecificationArgs{
					PredefinedMetricType: pulumi.String("ECSServiceAverageCPUUtilization"),
				},
				TargetValue:      pulumi.Float64(30.0),
				ScaleInCooldown:  pulumi.Int(60),
				ScaleOutCooldown: pulumi.Int(60),
			},
		})
		if err != nil {
			return err
		}

		// The URL at which the container's HTTP endpoint will be available
		ctx.Export("url", pulumi.Sprintf("http://%s", loadbalancer.LoadBalancer.DnsName()))
		return nil
	})
}
